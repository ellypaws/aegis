import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Routes, Route, useNavigate, useLocation, matchPath } from "react-router-dom";
import { intersect, cn } from "./lib/utils";
import { UI } from "./constants";
import { MOCK_GUILD } from "./data/mock";
import type { DiscordUser, Guild, Post, ViewMode } from "./types";
import { DiagonalSlitHeader } from "./components/DiagonalSlitHeader";
import { LoginModal } from "./components/LoginModal";
import { AuthorPanel, type AuthorPanelPostInput } from "./components/AuthorPanel";
import { TopBar } from "./components/TopBar";
import { MainGalleryView } from "./components/MainGalleryView";
import { PostDetailView } from "./components/PostDetailView";
import { RightSidebar } from "./components/RightSidebar";
import { ProfileSidebar } from "./components/ProfileSidebar";
import { NotFound } from "./pages/NotFound";
import { Settings } from "./pages/Settings";

function App() {
  const [user, setUser] = useState<DiscordUser | null>(null);
  const [guild] = useState<Guild>(MOCK_GUILD);
  const [loginOpen, setLoginOpen] = useState(false);
  const [posts, setPosts] = useState<Post[]>([]);
  const [loading, setLoading] = useState(false);
  const [initialLoading, setInitialLoading] = useState(true);

  const navigate = useNavigate();
  const location = useLocation();

  const [tagFilter, setTagFilter] = useState<string | null>(null);
  const [q, setQ] = useState("");
  const [transitionRect, setTransitionRect] = useState<DOMRect | null>(null);
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
    if (location.pathname === "/settings") return "gallery"; // Keep gallery view style for settings roughly
    return "not-found";
  }, [location.pathname, selectedId]);

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
        const base64Url = token.split('.')[1];
        const base64 = base64Url.replace(/-/g, '+').replace(/_/g, '/');
        const jsonPayload = decodeURIComponent(window.atob(base64).split('').map(function (c) {
          return '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2);
        }).join(''));

        const claims = JSON.parse(jsonPayload);
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
          roles: claims.roles ? claims.roles.map((r: any) => ({
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
            flags: r.flags
          })) : [],
          isAdmin: claims.adm,
          isAuthor: claims.adm
        };
        setUser(u);
      } catch (e) {
        console.error("Failed to parse JWT", e);
        localStorage.removeItem("jwt");
      }
    }
  }, []);

  // Fetch posts
  const [page, setPage] = useState(1);
  const [hasMore, setHasMore] = useState(true);
  const limit = 9;

  const loadPosts = useCallback((pageNum: number, reset: boolean = false, sort: SortMode = sortMode) => {
    setLoading(true);
    const token = localStorage.getItem("jwt");
    const headers: Record<string, string> = {};
    if (token) {
      headers["Authorization"] = `Bearer ${token}`;
    }

    // If reset and we need all currently-visible items, compute the right limit
    const fetchLimit = reset && pageNum === 1 ? Math.max(limit, posts.length) : limit;

    fetch(`/posts?page=${pageNum}&limit=${fetchLimit}&sort=${sort}`, { headers })
      .then(res => res.json())
      .then(data => {
        const items: Post[] = Array.isArray(data) ? data : [];
        const newPosts = reset ? items : [...posts, ...items];
        setPosts(newPosts);
        // Update cache for this sort mode
        sortCacheRef.current[sort] = newPosts;
        setHasMore(items.length === fetchLimit);
        setLoading(false);
        setInitialLoading(false);
      })
      .catch(err => {
        console.error("Failed to fetch posts", err);
        setLoading(false);
        setInitialLoading(false);
      });
  }, [sortMode, posts.length]);

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

  const selected = useMemo(() => posts?.find((p) => p.postKey === selectedId) ?? null, [posts, selectedId]);

  // If we have a selectedId but it's not in the loaded posts list, fetch it individually
  useEffect(() => {
    if (!selectedId || selected) return;
    const token = localStorage.getItem("jwt");
    const headers: Record<string, string> = {};
    if (token) headers["Authorization"] = `Bearer ${token}`;

    fetch(`/posts/${selectedId}`, { headers })
      .then(res => {
        if (!res.ok) return null;
        return res.json();
      })
      .then(post => {
        if (post) {
          setPosts(prev => {
            // Avoid duplicates
            if (prev.some(p => p.postKey === post.postKey)) return prev;
            return [post, ...prev];
          });
        }
      })
      .catch(err => console.error("Failed to fetch individual post", err));
  }, [selectedId, selected]);

  const viewerRoleIds = useMemo(() => (user?.roles ?? []).map((r) => r.id), [user]);

  const filteredPosts = useMemo(() => {
    const base = posts.slice();
    const byTag = base; // Tag filtering logic if tags field exists in future
    const s = q.trim().toLowerCase();
    if (!s) return byTag;
    return byTag.filter((p) => {
      const hay = `${p.title ?? ""} ${(p.description ?? "").slice(0, 120)}`.toLowerCase();
      return hay.includes(s);
    });
  }, [posts, tagFilter, q]);

  const canAccessPost = useMemo(() => {
    return (p: Post) => {
      if (user?.isAdmin) return true; // Admin/Author access
      const postRoleIds = p.allowedRoles.map(r => r.id);
      if (postRoleIds.length === 0) return true;
      return intersect(postRoleIds, viewerRoleIds);
    };
  }, [user, viewerRoleIds]);

  async function handleCreate(postInput: {
    title: string;
    description: string;
    allowedRoleIds: string[];
    channelIds: string[];
    image: File;
    thumbnail?: File;
    postDate: string;
    focusX?: number;
    focusY?: number;
  }) {
    const formData = new FormData();
    formData.append("title", postInput.title);
    formData.append("description", postInput.description);
    formData.append("roles", postInput.allowedRoleIds.join(","));
    formData.append("channels", postInput.channelIds.join(","));
    formData.append("image", postInput.image);
    formData.append("postDate", postInput.postDate);
    formData.append("focusX", String(postInput.focusX ?? 50));
    formData.append("focusY", String(postInput.focusY ?? 50));

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

  async function handleUpdate(postKey: string, postInput: AuthorPanelPostInput & { image?: File }) {
    const formData = new FormData();
    formData.append("title", postInput.title);
    formData.append("description", postInput.description);
    formData.append("roles", postInput.allowedRoleIds.join(","));
    formData.append("channels", postInput.channelIds.join(","));
    formData.append("focusX", String(postInput.focusX ?? 50));
    formData.append("focusY", String(postInput.focusY ?? 50));
    if (postInput.image) {
      formData.append("image", postInput.image);
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
            const pick = filteredPosts[Math.floor(Math.random() * Math.max(1, filteredPosts.length))];
            if (pick) {
              navigate(`/post/${pick.postKey}`);
            }
          }}
        />

        {view !== "not-found" && (
          <TopBar
            guild={guild}
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
          />
        )}

        {user?.isAdmin ? (
          <div className="mt-6">
            <AuthorPanel
              user={user}
              onCreate={handleCreate}
              editingPost={editingPost}
              onUpdate={handleUpdate}
              onCancelEdit={() => setEditingPost(null)}
            />
          </div>
        ) : null}

        <div className="mt-6 flex flex-col gap-4 lg:flex-row relative items-start">
          {/* Left: MAIN */}
          <div className={cn("transition-all duration-500 w-full lg:w-[calc(100%-22rem)]")}>
            <Routes>
              <Route path="/settings" element={<Settings user={user} />} />
              <Route
                path="/"
                element={
                  <MainGalleryView
                    posts={filteredPosts}
                    selectedId={selectedId}
                    canAccessPost={canAccessPost}
                    onOpenPost={(id, rect) => {
                      setTransitionRect(rect || null);
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
                    transitionRect={transitionRect}
                    user={user}
                  />
                }
              />
              <Route path="*" element={<NotFound />} />
            </Routes>
          </div>

          {/* Right: SIDEBAR - Do not show on 404? or default? */}
          {view !== "not-found" && (
            <div
              className={cn(
                "hidden w-80 lg:block"
              )}
            >
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
                    onEditPost={(post) => {
                      setEditingPost(post);
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
  );
}

export default App;
